$(function () {
    (async () => {
        $('html').addClass ( 'dom-loaded' );
    
        param_str = window.location.search.substring(1);
        params = new URLSearchParams(param_str);
        
        console.log()

        const tags_resp = await ky.get('/eve-routes/api/list_tags').json();
        const systems_resp = await ky.get('/eve-routes/api/list_systems').json();

        // tags_resp = {
        //     Items: ['b', 'a', 'c', 'Q']
        // }

        tags = tags_resp.Items.slice();
        tags.sort(function (x, y){
            xLower = isLowerCase(x.substring(0, 1))
            yLower = isLowerCase(y.substring(0, 1)) 
            if (xLower && yLower) {
                return x.localeCompare(y);
            }

            if (xLower) {
                return -1;
            }

            if (yLower) {
                return 1;
            }

            return x.localeCompare(y);
        });

        $('input#from_systems').val(params.get('from_systems'));
        $('input#to_system').val(params.get('to_system'));
        $('input#avoid_systems').val(params.get('avoid_systems'));
        
        tags.forEach(tag => {
            to_tag_elem = $("<option>").attr('value', tag).text(tag);
            if (params.get('to_tag') === tag) {
                to_tag_elem = to_tag_elem.attr("selected", "selected");
            }
            $('select#to_tag').append(to_tag_elem[0]);

            avoid_tag_elem = $("<option>").attr('value', tag).text(tag);
            if (params.getAll('avoid_tags').includes(tag)) {
                avoid_tag_elem = avoid_tag_elem.attr("selected", "selected");
            }
            $('select#avoid_tags').append(avoid_tag_elem[0]);

            prefer_tag_elem = $("<option>").attr('value', tag).text(tag);
            if (params.getAll('prefer_not_tags').includes(tag)) {
                prefer_tag_elem = prefer_tag_elem.attr("selected", "selected");
            }
            $('select#prefer_not_tags').append(prefer_tag_elem[0]);
        });

        if (param_str !== "") {
            await getRoutes();
        }
    })();
});

async function getRoutes() {
    body = {
        from_systems: $("input#from_systems").val().split(",").map(e => {return e.trim()}),
        to_system: $("input#to_system").val(),
        to_tag: $("select#to_tag").val(),
        avoid_systems: $("input#avoid_systems").val().split(",").map(e => {return e.trim()}),
        avoid_tags: $("select#avoid_tags").val(),
        prefer_not_tags: $("select#prefer_not_tags").val(),
    }

    routes_div = $("div#routes");
    routes_list_div = $("div#routes-list");
    routes_list_div.hide();

    console.log(body);
    const resp = await ky.post('/eve-routes/api/get_routes', {json: body}).json();

    routes_list_div.empty();
    resp.Routes.forEach(rt => {
        route = $("<ol>").addClass('route');
        rt.forEach(sd => {
            step = $("<li>").addClass(sd.sec_status).text(sd.name);
            route.append(step);
        });
        routes_list_div.append(route);
        routes_list_div.append()
    });

    routes_list_div.show();
    routes_div.show();
}

function isLowerCase(str)
{
    return str == str.toLowerCase() && str != str.toUpperCase();
}