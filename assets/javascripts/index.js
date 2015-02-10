$(document).ready(function() {

  //Navigation

  $('#nav-toggle').sidr({
    name: 'main-nav',
    source: '#navigation',
    side: 'right'
  });

  $('#sidr-id-nav-close').click(function () {
    $.sidr('close', 'main-nav');
  });

  $(window).touchwipe({
    wipeRight: function() {
      $.sidr('close', 'main-nav');
    },
    wipeLeft: function() {
      $.sidr('open', 'main-nav');
    },
    preventDefaultEvents: false
  });


  var $header = $('header');
      height = $('header').height();
      size = window.matchMedia('all and (min-width: 650px)');
  
  if(size.matches) {
    $(window).scroll(function() {
      if( $(this).scrollTop() > height ) {
        $header.addClass('small');
      } else {
        $header.removeClass('small');
      }
    });
  }
});